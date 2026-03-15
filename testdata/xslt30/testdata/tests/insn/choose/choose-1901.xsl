<?xml version="1.0" ?>
<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs"
version="2.0">
<!--  Test "if" - true branch
-->
<?spec xpath#id-conditionals?>
<xsl:template match="doc">
<out>
<xsl:value-of select="if (xs:integer(three/@val) > xs:integer(two/@val))
then true/@val else false/@val"/>
</out>
</xsl:template>
</xsl:stylesheet>
