<?xml version="1.0"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://ns.example.com/strip-type"
	exclude-result-prefixes="xs my" input-type-annotations="strip" version="2.0">

	<!-- Stylesheet is used for import / include -->

	<t:template name="temp2">
		<t:text>&#10;</t:text>
		<imported2>
			<t:text>&#10;</t:text>
			<i1>
				<t:value-of select="elem-duration instance of element(*, xs:duration)"/>
				<t:value-of select="elem-duration instance of element(*, xs:untyped)"/>
			</i1>
			<t:text>&#10;</t:text>
			<i2>
				<t:value-of select="elem-QName instance of element(*, xs:QName)"/>
				<t:value-of select="elem-QName instance of element(*, xs:untyped)"/>
			</i2>
			<t:text>&#10;</t:text>
			<i3>
				<t:value-of select="elem-string instance of element(*, xs:string)"/>
				<t:value-of select="elem-string instance of element(*, xs:untyped)"/>
			</i3>
			<t:text>&#10;</t:text>
			<i4>
				<t:value-of select="elem-NCName instance of element(*, xs:NCName)"/>
				<t:value-of select="elem-NCName instance of element(*, xs:untyped)"/>
			</i4>
			<t:text>&#10;</t:text>
			<i5>
				<t:value-of select="elem-NMTOKEN instance of element(*, xs:NMTOKEN)"/>
				<t:value-of select="elem-NMTOKEN instance of element(*, xs:untyped)"/>
			</i5>
			<t:text>&#10;</t:text>
			<i6>
				<t:value-of
					select="elem-nonPositiveInteger instance of element(*, xs:nonPositiveInteger)"/>
				<t:value-of select="elem-nonPositiveInteger instance of element(*, xs:untyped)"/>
			</i6>
			<t:text>&#10;</t:text>
			<i7>
				<t:value-of select="elem-long instance of element(*, xs:long)"/>
				<t:value-of select="elem-long instance of element(*, xs:untyped)"/>
			</i7>
			<t:text>&#10;</t:text>
			<i8>
				<t:value-of select="simple-derived instance of element(*, my:partNumberType)"/>
				<t:value-of select="simple-derived instance of element(*, xs:untyped)"/>
			</i8>
			<t:text>&#10;</t:text>
			<i9>
				<t:value-of select="complex-derived instance of element(*, my:personType)"/>
				<t:value-of select="complex-derived instance of element(*, xs:untyped)"/>
			</i9>
			<t:text>&#10;</t:text>
			<i10>
				<t:value-of select="complex-derived/familyName instance of element(*, xs:string)"/>
				<t:value-of select="complex-derived/familyName instance of element(*, xs:untyped)"/>
			</i10>
			<t:text>&#10;</t:text>
			<i11>
				<t:value-of select="complex-derived/@age instance of attribute(*, xs:integer)"/>
				<t:value-of select="complex-derived/@age instance of attribute(*, xs:untypedAtomic)"
				/>
			</i11>
			<t:text>&#10;</t:text>
			<i12>
				<t:value-of select="list-derived instance of element(*, my:myListType)"/>
				<t:value-of select="list-derived instance of element(*, xs:untyped)"/>
			</i12>
			<t:text>&#10;</t:text>
			<i13>
				<t:value-of select="union-derived[1] instance of element(*, my:partIntegerUnion)"/>
				<t:value-of select="union-derived[1] instance of element(*, xs:untyped)"/>
			</i13>
			<t:text>&#10;</t:text>
		</imported2>
	</t:template>

</t:transform>
