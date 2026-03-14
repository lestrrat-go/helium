<?xml version="1.0"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://ns.example.com/strip-type"
	exclude-result-prefixes="xs my" input-type-annotations="strip" version="2.0">

	<!-- Purpose: Test that type annotations are stripped from the initial source tree if an imported, 
		an included and the primary stylesheet have @input-type-annotations="strip".-->

	<t:import href="import2_strip.xsl"/>
	<t:include href="import_strip.xsl"/>

	<t:import-schema namespace="http://ns.example.com/strip-type"
		schema-location="varietyOfTypes.xsd"/>

	<t:template match="/my:doc">
		<out>
			<primary>
				<E1>
					<t:value-of select="elem-anyURI instance of element(*, xs:anyURI)"/>
					<t:value-of select="elem-anyURI instance of element(*, xs:untyped)"/>
				</E1>
				<E2>
					<t:value-of select="elem-boolean instance of element(*, xs:boolean)"/>
					<t:value-of select="elem-boolean instance of element(*, xs:untyped)"/>
				</E2>
				<E3>
					<t:value-of select="elem-date instance of element(*, xs:date)"/>
					<t:value-of select="elem-date instance of element(*, xs:untyped)"/>
				</E3>
				<E4>
					<t:value-of select="elem-double instance of element(*, xs:double)"/>
					<t:value-of select="elem-double instance of element(*, xs:untyped)"/>
				</E4>
				<E5>
					<t:value-of select="elem-duration instance of element(*, xs:duration)"/>
					<t:value-of select="elem-duration instance of element(*, xs:untyped)"/>
				</E5>
				<E6>
					<t:value-of select="elem-QName instance of element(*, xs:QName)"/>
					<t:value-of select="elem-QName instance of element(*, xs:untyped)"/>
				</E6>
				<E7>
					<t:value-of select="elem-string instance of element(*, xs:string)"/>
					<t:value-of select="elem-string instance of element(*, xs:untyped)"/>
				</E7>
				<E8>
					<t:value-of select="elem-NCName instance of element(*, xs:NCName)"/>
					<t:value-of select="elem-NCName instance of element(*, xs:untyped)"/>
				</E8>
				<E9>
					<t:value-of select="elem-NMTOKEN instance of element(*, xs:NMTOKEN)"/>
					<t:value-of select="elem-NMTOKEN instance of element(*, xs:untyped)"/>
				</E9>
				<E10>
					<t:value-of select="elem-int instance of element(*, xs:int)"/>
					<t:value-of select="elem-int instance of element(*, xs:untyped)"/>
				</E10>
				<E11>
					<t:value-of select="elem-nonPositiveInteger instance of element(*, xs:nonPositiveInteger)"/>
					<t:value-of select="elem-nonPositiveInteger instance of element(*, xs:untyped)"/>
				</E11>
				<E12>
					<t:value-of select="elem-long instance of element(*, xs:long)"/>
					<t:value-of select="elem-long instance of element(*, xs:untyped)"/>
				</E12>
				<E13>
					<t:value-of select="elem-unsignedLong instance of element(*, xs:unsignedLong)"/>
					<t:value-of select="elem-unsignedLong instance of element(*, xs:untyped)"/>
				</E13>
				<E14>
					<t:value-of select="simple-derived instance of element(*, my:partNumberType)"/>
					<t:value-of select="simple-derived instance of element(*, xs:untyped)"/>
				</E14>
				<E15>
					<t:value-of select="complex-derived instance of element(*, my:personType)"/>
					<t:value-of select="complex-derived instance of element(*, xs:untyped)"/>
				</E15>
				<E16>
					<t:value-of select="complex-derived/firstName instance of element(*, xs:string)"/>
					<t:value-of	select="complex-derived/firstName instance of element(*, xs:untyped)"/>
				</E16>
				<E17>
					<t:value-of	select="complex-derived/familyName instance of element(*, xs:string)"/>
					<t:value-of	select="complex-derived/familyName instance of element(*, xs:untyped)"/>
				</E17>
				<E19>
					<t:value-of select="complex-derived/@age instance of attribute(*, xs:integer)"/>
					<t:value-of	select="complex-derived/@age instance of attribute(*, xs:untypedAtomic)"/>
				</E19>
				<E20>
					<t:value-of select="list-builtin instance of element(*, xs:NMTOKENS)"/>
					<t:value-of select="list-builtin instance of element(*, xs:untyped)"/>
				</E20>
				<E21>
					<t:value-of select="list-derived instance of element(*, my:myListType)"/>
					<t:value-of select="list-derived instance of element(*, xs:untyped)"/>
				</E21>
				<E22>
					<t:value-of	select="union-derived[1] instance of element(*, my:partIntegerUnion)"/>
					<t:value-of select="union-derived[1] instance of element(*, xs:untyped)"/>
				</E22>
				<E23>
					<t:value-of	select="union-derived[2] instance of element(*, my:partIntegerUnion)"/>
					<t:value-of select="union-derived[2] instance of element(*, xs:untyped)"/>
				</E23>
			</primary>
			<t:call-template name="temp"/>
			<t:call-template name="temp2"/>
		</out>
	</t:template>

</t:transform>
