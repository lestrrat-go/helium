<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
				xmlns:xs="http://www.w3.org/2001/XMLSchema" 
				xmlns:my="http://ns.example.com/strip-type"
				exclude-result-prefixes="xs my"
				input-type-annotations="preserve" 
				version="2.0">

  <!-- Stylesheet is used for import / include -->

  <xsl:template name="temp">
  	<xsl:text>&#10;</xsl:text>
  		<imported>
  			<xsl:text>&#10;</xsl:text>
    		<I1><xsl:value-of select="elem-anyURI instance of element(*, xs:anyURI)" />
    		    <xsl:value-of select="elem-anyURI instance of element(*, xs:untyped)" /></I1><xsl:text>&#10;</xsl:text>
			<I2><xsl:value-of select="elem-boolean instance of element(*, xs:boolean)" />
				<xsl:value-of select="elem-boolean instance of element(*, xs:untyped)" /></I2><xsl:text>&#10;</xsl:text>
			<I3><xsl:value-of select="elem-date instance of element(*, xs:date)" />
				<xsl:value-of select="elem-date instance of element(*, xs:untyped)" /></I3><xsl:text>&#10;</xsl:text>
			<I4><xsl:value-of select="elem-double instance of element(*, xs:double)" />
				<xsl:value-of select="elem-double instance of element(*, xs:untyped)" /></I4><xsl:text>&#10;</xsl:text>
			<I5><xsl:value-of select="elem-duration instance of element(*, xs:duration)" />
				<xsl:value-of select="elem-duration instance of element(*, xs:untyped)" /></I5><xsl:text>&#10;</xsl:text>
			<I6><xsl:value-of select="elem-QName instance of element(*, xs:QName)" />
				<xsl:value-of select="elem-QName instance of element(*, xs:untyped)" /></I6><xsl:text>&#10;</xsl:text>
			<I7><xsl:value-of select="elem-string instance of element(*, xs:string)" />
				<xsl:value-of select="elem-string instance of element(*, xs:untyped)" /></I7><xsl:text>&#10;</xsl:text>
			<I8><xsl:value-of select="elem-NCName instance of element(*, xs:NCName)" />
				<xsl:value-of select="elem-NCName instance of element(*, xs:untyped)" /></I8><xsl:text>&#10;</xsl:text>
			<I9><xsl:value-of select="elem-NMTOKEN instance of element(*, xs:NMTOKEN)" />
				<xsl:value-of select="elem-NMTOKEN instance of element(*, xs:untyped)" /></I9><xsl:text>&#10;</xsl:text>
			<I10><xsl:value-of select="elem-int instance of element(*, xs:int)" />
				 <xsl:value-of select="elem-int instance of element(*, xs:untyped)" /></I10><xsl:text>&#10;</xsl:text>
			<I11><xsl:value-of select="elem-nonPositiveInteger instance of element(*, xs:nonPositiveInteger)" />
				 <xsl:value-of select="elem-nonPositiveInteger instance of element(*, xs:untyped)" /></I11><xsl:text>&#10;</xsl:text>
			<I12><xsl:value-of select="elem-long instance of element(*, xs:long)" />
				 <xsl:value-of select="elem-long instance of element(*, xs:untyped)" /></I12><xsl:text>&#10;</xsl:text>
			<I13><xsl:value-of select="elem-unsignedLong instance of element(*, xs:unsignedLong)" />
				 <xsl:value-of select="elem-unsignedLong instance of element(*, xs:untyped)" /></I13><xsl:text>&#10;</xsl:text>
			<I14><xsl:value-of select="simple-derived instance of element(*, my:partNumberType)" />
				 <xsl:value-of select="simple-derived instance of element(*, xs:untyped)" /></I14><xsl:text>&#10;</xsl:text>
			<I15><xsl:value-of select="complex-derived instance of element(*, my:personType)" />
				 <xsl:value-of select="complex-derived instance of element(*, xs:untyped)" /></I15><xsl:text>&#10;</xsl:text>
			<I16><xsl:value-of select="complex-derived/firstName instance of element(*, xs:string)" />
				 <xsl:value-of select="complex-derived/firstName instance of element(*, xs:untyped)" /></I16><xsl:text>&#10;</xsl:text>
			<I17><xsl:value-of select="complex-derived/familyName instance of element(*, xs:string)" />
				 <xsl:value-of select="complex-derived/familyName instance of element(*, xs:untyped)" /></I17><xsl:text>&#10;</xsl:text>
			<I19><xsl:value-of select="complex-derived/@age instance of attribute(*, xs:integer)" />
				 <xsl:value-of select="complex-derived/@age instance of attribute(*, xs:untypedAtomic)" /></I19><xsl:text>&#10;</xsl:text>
			<I20><xsl:value-of select="list-builtin instance of element(*, xs:NMTOKENS)" />
				 <xsl:value-of select="list-builtin instance of element(*, xs:untyped)" /></I20><xsl:text>&#10;</xsl:text>
			<I21><xsl:value-of select="list-derived instance of element(*, my:myListType)" />
				 <xsl:value-of select="list-derived instance of element(*, xs:untyped)" /></I21><xsl:text>&#10;</xsl:text>
			<I22><xsl:value-of select="union-derived[1] instance of element(*, my:partIntegerUnion)"/>
				 <xsl:value-of select="union-derived[1] instance of element(*, xs:untyped)"/></I22><xsl:text>&#10;</xsl:text>
			<I23><xsl:value-of select="union-derived[2] instance of element(*, my:partIntegerUnion)"/>
				 <xsl:value-of select="union-derived[2] instance of element(*, xs:untyped)"/></I23><xsl:text>&#10;</xsl:text>
  		</imported>
  </xsl:template>

</xsl:stylesheet>