<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION:  4.1 Node Set Functions, 2.4 Predicates. -->
<!-- PURPOSE:  Test of 'position()' in 2nd predicate. -->
<xsl:template match="doc">
<out>
<xsl:value-of select="*[@test][position()=3]/num"/>
</out>
</xsl:template>
</xsl:stylesheet>