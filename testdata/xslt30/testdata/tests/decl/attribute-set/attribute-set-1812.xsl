<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0">

 <?spec xslt#attribute-sets?>
<!-- check use of attribute sets with a computed element name -->
<!-- Saxon bug 1244273 -->

	
	<xsl:attribute-set name="foobar">
		<xsl:attribute name="foo">bar</xsl:attribute>
	</xsl:attribute-set>
	
	<xsl:variable name="element_name">Data</xsl:variable>
	
	<xsl:template match="/">
		<!-- It seems Saxon has a problem with the following line... -->
		<xsl:element name="{$element_name}" use-attribute-sets="foobar">
		<!-- whereas this line will work as expected... --> 
		<!--xsl:element name="Data" use-attribute-sets="foobar" -->
			<xsl:copy-of select="."/>
		</xsl:element>
	</xsl:template>

</xsl:stylesheet>
