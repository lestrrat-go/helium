<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xslt#character-maps?>

<!-- tests use of a character map in xsl:result-document -->

<xsl:stylesheet version="2.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<xsl:character-map name="c" >
  <xsl:output-character character="«" string="{{"/>
  <xsl:output-character character="»" string="}}"/>
</xsl:character-map>

<xsl:template match="/">
  <xsl:result-document use-character-maps="c" encoding="iso-8859-1">
	<out><xsl:text>[«=»]</xsl:text></out>
  </xsl:result-document>		
</xsl:template>

</xsl:stylesheet>