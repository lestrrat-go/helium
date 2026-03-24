<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xslt#character-maps?>



<!-- tests generation of special characters in replacement string
     with HTML output method (both text and attribute nodes) -->

<xsl:stylesheet version="2.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<xsl:character-map name="htmlTrick" >
  <xsl:output-character character="«" string="&lt;%"/>
  <xsl:output-character character="»" string="%&gt;"/>
  <xsl:output-character character="§" string='"'/>
</xsl:character-map>

<xsl:output method="html" indent="yes" encoding="iso-8859-1"
    use-character-maps="htmlTrick"
/>

<xsl:template match="/">
<html>
    <body>value='«= §id§ + idValue »'
        <setProperty value='«= §id§ + idValue »'/>
    </body>
</html>
</xsl:template>

<xsl:template name="GetInput">
<html>
    <body>value='«= §id§ + idValue »'
        <setProperty value='«= §id§ + idValue »'/>
    </body>
</html>
</xsl:template>

</xsl:stylesheet>