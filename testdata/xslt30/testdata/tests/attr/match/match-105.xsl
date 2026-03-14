<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <!-- Test of pattern matching with function key() with variable reference as argument -->

<xsl:variable name="temp" select="'elem_d'"/>

<xsl:key name="kname" match="/doc/*" use="string(@id)"/>

<xsl:template match="doc">
	<xsl:apply-templates/>
</xsl:template>

<xsl:template match="key('kname', $temp)">
  <t>
	<xsl:value-of select="name(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>

<xsl:template match="*"/>

</xsl:stylesheet>
