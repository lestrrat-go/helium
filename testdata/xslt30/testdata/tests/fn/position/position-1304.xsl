<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION:  4.1 Node Set Functions, 2.4 Predicates. -->
<!-- PURPOSE:  Test of positional indexing when used with xsl:key. -->
<xsl:key name="k" match="a" use="@test"/>
<xsl:key name="k2" match="a" use="num"/>
<xsl:template match="doc">
<out>
<xsl:value-of select="key('k','true')[4]/num"/>
</out>
</xsl:template>
</xsl:stylesheet>