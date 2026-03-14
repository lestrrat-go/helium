<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">

	<!--          Test with global xsl:variable where @select has an xs:untypedAtomic
	 			value of a typed value the same in @as obtained at run time, @as is a 
	 			built-in derived atomic type. Verify the variable is of the type specified 
	 			in @as. -->


	<xsl:variable name="v1" select="/doc-schemaas/elem-anyURI" as="xs:anyURI"/>
	<xsl:variable name="v2" select="/doc-schemaas/elem-base64Binary" as="xs:base64Binary"/>
	<xsl:variable name="v3" select="/doc-schemaas/elem-boolean" as="xs:boolean"/>
	<xsl:variable name="v4" select="/doc-schemaas/elem-date" as="xs:date"/>
	<xsl:variable name="v5" select="/doc-schemaas/elem-dateTime" as="xs:dateTime"/>
	<xsl:variable name="v6" select="/doc-schemaas/elem-decimal" as="xs:decimal"/>
	<xsl:variable name="v7" select="/doc-schemaas/elem-double" as="xs:double"/>
	<xsl:variable name="v8" select="/doc-schemaas/elem-duration" as="xs:duration"/>
	<xsl:variable name="v9" select="/doc-schemaas/elem-float" as="xs:float"/>
	<xsl:variable name="v10" select="/doc-schemaas/elem-gDay" as="xs:gDay"/>
	<xsl:variable name="v11" select="/doc-schemaas/elem-gMonth" as="xs:gMonth"/>
	<xsl:variable name="v12" select="/doc-schemaas/elem-gMonthDay" as="xs:gMonthDay"/>
	<xsl:variable name="v13" select="/doc-schemaas/elem-gYear" as="xs:gYear"/>
	<xsl:variable name="v14" select="/doc-schemaas/elem-gYearMonth" as="xs:gYearMonth"/>
	<xsl:variable name="v15" select="/doc-schemaas/elem-hexBinary" as="xs:hexBinary"/>
	<xsl:variable name="v16" select="/doc-schemaas/elem-integer" as="xs:integer"/>
	<xsl:variable name="v17"
		select="if (/doc-schemaas/elem-QName instance of element(*,xs:untyped)) 
		        then resolve-QName(/doc-schemaas/elem-QName,/doc-schemaas/elem-QName) 
		        else /doc-schemaas/elem-QName"
		as="xs:QName"/>
	<xsl:variable name="v18" select="/doc-schemaas/elem-string" as="xs:string"/>
	<xsl:variable name="v19" select="/doc-schemaas/elem-time" as="xs:time"/>
	<xsl:variable name="v20" select="/doc-schemaas/elem-normalizedString" as="xs:normalizedString"/>
	<xsl:variable name="v21" select="/doc-schemaas/elem-token" as="xs:token"/>
	<xsl:variable name="v22" select="/doc-schemaas/elem-language" as="xs:language"/>
	<xsl:variable name="v23" select="/doc-schemaas/elem-Name" as="xs:Name"/>
	<xsl:variable name="v24" select="/doc-schemaas/elem-NCName" as="xs:NCName"/>
	<xsl:variable name="v25" select="/doc-schemaas/elem-NMTOKEN" as="xs:NMTOKEN"/>
	<xsl:variable name="v26" select="/doc-schemaas/elem-int" as="xs:int"/>
	<xsl:variable name="v27" select="/doc-schemaas/elem-nonPositiveInteger"
		as="xs:nonPositiveInteger"/>
	<xsl:variable name="v28" select="/doc-schemaas/elem-positiveInteger" as="xs:positiveInteger"/>
	<xsl:variable name="v29" select="/doc-schemaas/elem-nonNegativeInteger"
		as="xs:nonNegativeInteger"/>
	<xsl:variable name="v30" select="/doc-schemaas/elem-negativeInteger" as="xs:negativeInteger"/>
	<xsl:variable name="v31" select="/doc-schemaas/elem-long" as="xs:long"/>
	<xsl:variable name="v32" select="/doc-schemaas/elem-unsignedLong" as="xs:unsignedLong"/>
	<xsl:variable name="v33" select="/doc-schemaas/elem-unsignedInt" as="xs:unsignedInt"/>
	<xsl:variable name="v34" select="/doc-schemaas/elem-short" as="xs:short"/>
	<xsl:variable name="v35" select="/doc-schemaas/elem-unsignedShort" as="xs:unsignedShort"/>
	<xsl:variable name="v36" select="/doc-schemaas/elem-byte" as="xs:byte"/>
	<xsl:variable name="v37" select="/doc-schemaas/elem-unsignedByte" as="xs:unsignedByte"/>
	<xsl:variable name="v38" select="/doc-schemaas/elem-ID-attr/@attr1" as="xs:ID"/>
	<xsl:variable name="v39" select="/doc-schemaas/elem-IDREF-attr/@attr1" as="xs:IDREF"/>


	<xsl:template match="/doc-schemaas">
		<out>
			<var1>
				<xsl:value-of select="$v1 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v1 instance of xs:anyURI"/>
			</var1>
			<var2>
				<xsl:value-of select="$v2 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v2 instance of xs:base64Binary"/>
			</var2>
			<var3>
				<xsl:value-of select="$v3 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v3 instance of xs:boolean"/>
			</var3>
			<var4>
				<xsl:value-of select="$v4 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v4 instance of xs:date"/>
			</var4>
			<var5>
				<xsl:value-of select="$v5 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v5 instance of xs:dateTime"/>
			</var5>
			<var6>
				<xsl:value-of select="$v6 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v6 instance of xs:decimal"/>
			</var6>
			<var7>
				<xsl:value-of select="$v7 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v7 instance of xs:double"/>
			</var7>
			<var8>
				<xsl:value-of select="$v8 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v8 instance of xs:duration"/>
			</var8>
			<var9>
				<xsl:value-of select="$v9 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v9 instance of xs:float"/>
			</var9>
			<var10>
				<xsl:value-of select="$v10 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v10 instance of xs:gDay"/>
			</var10>
			<var11>
				<xsl:value-of select="$v11 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v11 instance of xs:gMonth"/>
			</var11>
			<var12>
				<xsl:value-of select="$v12 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v12 instance of xs:gMonthDay"/>
			</var12>
			<var13>
				<xsl:value-of select="$v13 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v13 instance of xs:gYear"/>
			</var13>
			<var14>
				<xsl:value-of select="$v14 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v14 instance of xs:gYearMonth"/>
			</var14>
			<var15>
				<xsl:value-of select="$v15 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v15 instance of xs:hexBinary"/>
			</var15>
			<var16>
				<xsl:value-of select="$v16 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v16 instance of xs:integer"/>
			</var16>
			<var17>
				<xsl:value-of select="$v17 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v17 instance of xs:QName"/>
			</var17>
			<var18>
				<xsl:value-of select="$v18 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v18 instance of xs:string"/>
			</var18>
			<var19>
				<xsl:value-of select="$v19 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v19 instance of xs:time"/>
			</var19>
			<var20>
				<xsl:value-of select="$v20 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v20 instance of xs:normalizedString"/>
			</var20>
			<var21>
				<xsl:value-of select="$v21 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v21 instance of xs:token"/>
			</var21>
			<var22>
				<xsl:value-of select="$v22 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v22 instance of xs:language"/>
			</var22>
			<var23>
				<xsl:value-of select="$v23 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v23 instance of xs:Name"/>
			</var23>
			<var24>
				<xsl:value-of select="$v24 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v24 instance of xs:NCName"/>
			</var24>
			<var25>
				<xsl:value-of select="$v25 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v25 instance of xs:NMTOKEN"/>
			</var25>
			<var26>
				<xsl:value-of select="$v26 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v26 instance of xs:int"/>
			</var26>
			<var27>
				<xsl:value-of select="$v27 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v27 instance of xs:nonPositiveInteger"/>
			</var27>
			<var28>
				<xsl:value-of select="$v28 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v28 instance of xs:positiveInteger"/>
			</var28>
			<var29>
				<xsl:value-of select="$v29 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v29 instance of xs:nonNegativeInteger"/>
			</var29>
			<var30>
				<xsl:value-of select="$v30 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v30 instance of xs:negativeInteger"/>
			</var30>
			<var31>
				<xsl:value-of select="$v31 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v31 instance of xs:long"/>
			</var31>
			<var32>
				<xsl:value-of select="$v32 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v32 instance of xs:unsignedLong"/>
			</var32>
			<var33>
				<xsl:value-of select="$v33 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v33 instance of xs:unsignedInt"/>
			</var33>
			<var34>
				<xsl:value-of select="$v34 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v34 instance of xs:short"/>
			</var34>
			<var35>
				<xsl:value-of select="$v35 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v35 instance of xs:unsignedShort"/>
			</var35>
			<var36>
				<xsl:value-of select="$v36 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v36 instance of xs:byte"/>
			</var36>
			<var37>
				<xsl:value-of select="$v37 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v37 instance of xs:unsignedByte"/>
			</var37>
			<var38>
				<xsl:value-of select="$v38 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v38 instance of xs:ID"/>
			</var38>
			<var39>
				<xsl:value-of select="$v39 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$v39 instance of xs:IDREF"/>
			</var39>
		</out>
	</xsl:template>


</xsl:stylesheet>
