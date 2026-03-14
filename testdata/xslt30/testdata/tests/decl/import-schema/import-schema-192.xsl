<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" 
				xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				xmlns:xs="http://www.w3.org/2001/XMLSchema"
				exclude-result-prefixes="xs">
				
				<!-- test for using a union type -->
				
	<xsl:import-schema>
	  <xs:schema>
	  	<xs:simpleType name="dateUnion">
	  	  <xs:union memberTypes="xs:date xs:time xs:dateTime"/>
	  	</xs:simpleType>
	  </xs:schema>
	</xsl:import-schema>		
				
				
	<xsl:template name="main">
	<out>
        <xsl:variable name="d" as="dateUnion" select="xs:untypedAtomic('12:00:00')"/>
        <a><xsl:value-of select="$d"/></a>
        <b><xsl:value-of select="$d instance of xs:time"/></b>
        <c><xsl:value-of select="$d instance of dateUnion"/></c>
        <d><xsl:value-of select="format-time($d, '[H01]:[m01]')"/></d>
        <e><xsl:value-of select="'2008-11-14' cast as dateUnion"/></e>
    </out>
	</xsl:template>
	
</xsl:stylesheet>
