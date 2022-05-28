// Code generated by "stringer -type=MemoryKind -trimprefix=MemoryKind"; DO NOT EDIT.

package common

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[MemoryKindUnknown-0]
	_ = x[MemoryKindBool-1]
	_ = x[MemoryKindAddress-2]
	_ = x[MemoryKindString-3]
	_ = x[MemoryKindCharacter-4]
	_ = x[MemoryKindMetaType-5]
	_ = x[MemoryKindNumber-6]
	_ = x[MemoryKindArrayBase-7]
	_ = x[MemoryKindArrayLength-8]
	_ = x[MemoryKindDictionaryBase-9]
	_ = x[MemoryKindDictionarySize-10]
	_ = x[MemoryKindCompositeBase-11]
	_ = x[MemoryKindCompositeSize-12]
	_ = x[MemoryKindOptional-13]
	_ = x[MemoryKindNil-14]
	_ = x[MemoryKindVoid-15]
	_ = x[MemoryKindTypeValue-16]
	_ = x[MemoryKindPathValue-17]
	_ = x[MemoryKindCapabilityValue-18]
	_ = x[MemoryKindLinkValue-19]
	_ = x[MemoryKindStorageReferenceValue-20]
	_ = x[MemoryKindEphemeralReferenceValue-21]
	_ = x[MemoryKindInterpretedFunction-22]
	_ = x[MemoryKindHostFunction-23]
	_ = x[MemoryKindBoundFunction-24]
	_ = x[MemoryKindBigInt-25]
	_ = x[MemoryKindPrimitiveStaticType-26]
	_ = x[MemoryKindCompositeStaticType-27]
	_ = x[MemoryKindInterfaceStaticType-28]
	_ = x[MemoryKindVariableSizedStaticType-29]
	_ = x[MemoryKindConstantSizedStaticType-30]
	_ = x[MemoryKindDictionaryStaticType-31]
	_ = x[MemoryKindOptionalStaticType-32]
	_ = x[MemoryKindRestrictedStaticType-33]
	_ = x[MemoryKindReferenceStaticType-34]
	_ = x[MemoryKindCapabilityStaticType-35]
	_ = x[MemoryKindFunctionStaticType-36]
	_ = x[MemoryKindRawString-37]
	_ = x[MemoryKindAddressLocation-38]
	_ = x[MemoryKindBytes-39]
	_ = x[MemoryKindVariable-40]
	_ = x[MemoryKindValueToken-41]
	_ = x[MemoryKindSyntaxToken-42]
	_ = x[MemoryKindSpaceToken-43]
	_ = x[MemoryKindProgram-44]
	_ = x[MemoryKindIdentifier-45]
	_ = x[MemoryKindArgument-46]
	_ = x[MemoryKindBlock-47]
	_ = x[MemoryKindFunctionBlock-48]
	_ = x[MemoryKindParameter-49]
	_ = x[MemoryKindParameterList-50]
	_ = x[MemoryKindTransfer-51]
	_ = x[MemoryKindMembers-52]
	_ = x[MemoryKindTypeAnnotation-53]
	_ = x[MemoryKindDictionaryEntry-54]
	_ = x[MemoryKindFunctionDeclaration-55]
	_ = x[MemoryKindCompositeDeclaration-56]
	_ = x[MemoryKindInterfaceDeclaration-57]
	_ = x[MemoryKindEnumCaseDeclaration-58]
	_ = x[MemoryKindFieldDeclaration-59]
	_ = x[MemoryKindTransactionDeclaration-60]
	_ = x[MemoryKindImportDeclaration-61]
	_ = x[MemoryKindVariableDeclaration-62]
	_ = x[MemoryKindSpecialFunctionDeclaration-63]
	_ = x[MemoryKindPragmaDeclaration-64]
	_ = x[MemoryKindAssignmentStatement-65]
	_ = x[MemoryKindBreakStatement-66]
	_ = x[MemoryKindContinueStatement-67]
	_ = x[MemoryKindEmitStatement-68]
	_ = x[MemoryKindExpressionStatement-69]
	_ = x[MemoryKindForStatement-70]
	_ = x[MemoryKindIfStatement-71]
	_ = x[MemoryKindReturnStatement-72]
	_ = x[MemoryKindSwapStatement-73]
	_ = x[MemoryKindSwitchStatement-74]
	_ = x[MemoryKindWhileStatement-75]
	_ = x[MemoryKindBooleanExpression-76]
	_ = x[MemoryKindNilExpression-77]
	_ = x[MemoryKindStringExpression-78]
	_ = x[MemoryKindIntegerExpression-79]
	_ = x[MemoryKindFixedPointExpression-80]
	_ = x[MemoryKindArrayExpression-81]
	_ = x[MemoryKindDictionaryExpression-82]
	_ = x[MemoryKindIdentifierExpression-83]
	_ = x[MemoryKindInvocationExpression-84]
	_ = x[MemoryKindMemberExpression-85]
	_ = x[MemoryKindIndexExpression-86]
	_ = x[MemoryKindConditionalExpression-87]
	_ = x[MemoryKindUnaryExpression-88]
	_ = x[MemoryKindBinaryExpression-89]
	_ = x[MemoryKindFunctionExpression-90]
	_ = x[MemoryKindCastingExpression-91]
	_ = x[MemoryKindCreateExpression-92]
	_ = x[MemoryKindDestroyExpression-93]
	_ = x[MemoryKindReferenceExpression-94]
	_ = x[MemoryKindForceExpression-95]
	_ = x[MemoryKindPathExpression-96]
	_ = x[MemoryKindConstantSizedType-97]
	_ = x[MemoryKindDictionaryType-98]
	_ = x[MemoryKindFunctionType-99]
	_ = x[MemoryKindInstantiationType-100]
	_ = x[MemoryKindNominalType-101]
	_ = x[MemoryKindOptionalType-102]
	_ = x[MemoryKindReferenceType-103]
	_ = x[MemoryKindRestrictedType-104]
	_ = x[MemoryKindVariableSizedType-105]
	_ = x[MemoryKindPosition-106]
	_ = x[MemoryKindRange-107]
	_ = x[MemoryKindLast-108]
}

const _MemoryKind_name = "UnknownBoolAddressStringCharacterMetaTypeNumberArrayBaseArrayLengthDictionaryBaseDictionarySizeCompositeBaseCompositeSizeOptionalNilVoidTypeValuePathValueCapabilityValueLinkValueStorageReferenceValueEphemeralReferenceValueInterpretedFunctionHostFunctionBoundFunctionBigIntPrimitiveStaticTypeCompositeStaticTypeInterfaceStaticTypeVariableSizedStaticTypeConstantSizedStaticTypeDictionaryStaticTypeOptionalStaticTypeRestrictedStaticTypeReferenceStaticTypeCapabilityStaticTypeFunctionStaticTypeRawStringAddressLocationBytesVariableValueTokenSyntaxTokenSpaceTokenProgramIdentifierArgumentBlockFunctionBlockParameterParameterListTransferMembersTypeAnnotationDictionaryEntryFunctionDeclarationCompositeDeclarationInterfaceDeclarationEnumCaseDeclarationFieldDeclarationTransactionDeclarationImportDeclarationVariableDeclarationSpecialFunctionDeclarationPragmaDeclarationAssignmentStatementBreakStatementContinueStatementEmitStatementExpressionStatementForStatementIfStatementReturnStatementSwapStatementSwitchStatementWhileStatementBooleanExpressionNilExpressionStringExpressionIntegerExpressionFixedPointExpressionArrayExpressionDictionaryExpressionIdentifierExpressionInvocationExpressionMemberExpressionIndexExpressionConditionalExpressionUnaryExpressionBinaryExpressionFunctionExpressionCastingExpressionCreateExpressionDestroyExpressionReferenceExpressionForceExpressionPathExpressionConstantSizedTypeDictionaryTypeFunctionTypeInstantiationTypeNominalTypeOptionalTypeReferenceTypeRestrictedTypeVariableSizedTypePositionRangeLast"

var _MemoryKind_index = [...]uint16{0, 7, 11, 18, 24, 33, 41, 47, 56, 67, 81, 95, 108, 121, 129, 132, 136, 145, 154, 169, 178, 199, 222, 241, 253, 266, 272, 291, 310, 329, 352, 375, 395, 413, 433, 452, 472, 490, 499, 514, 519, 527, 537, 548, 558, 565, 575, 583, 588, 601, 610, 623, 631, 638, 652, 667, 686, 706, 726, 745, 761, 783, 800, 819, 845, 862, 881, 895, 912, 925, 944, 956, 967, 982, 995, 1010, 1024, 1041, 1054, 1070, 1087, 1107, 1122, 1142, 1162, 1182, 1198, 1213, 1234, 1249, 1265, 1283, 1300, 1316, 1333, 1352, 1367, 1381, 1398, 1412, 1424, 1441, 1452, 1464, 1477, 1491, 1508, 1516, 1521, 1525}

func (i MemoryKind) String() string {
	if i >= MemoryKind(len(_MemoryKind_index)-1) {
		return "MemoryKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _MemoryKind_name[_MemoryKind_index[i]:_MemoryKind_index[i+1]]
}
