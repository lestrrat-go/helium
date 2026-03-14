<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xslt#character-maps?>

<!-- tests generation of special characters in replacement string
     with text output method (both text and attribute nodes) -->

<xsl:stylesheet version="2.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<xsl:character-map name="c" >
  <xsl:output-character character="«" string="{{"/>
  <xsl:output-character character="»" string="}}"/>
</xsl:character-map>

<xsl:output method="text" encoding="iso-8859-1"
    use-character-maps="c"
/>

<xsl:template match="/">
	<xsl:text>[«=»]</xsl:text>
</xsl:template>

</xsl:stylesheet>