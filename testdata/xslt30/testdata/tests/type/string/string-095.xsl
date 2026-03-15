<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-literals?>
<!-- PURPOSE:   Test that quotes can be doubled in string literals (XPath 2.0) . -->
<xsl:template match="doc">
<out>
<a><xsl:value-of select="'He isn''t'"/></a>
<a><xsl:value-of select='"He said ""boo"""'/></a>
<b text="{'He isn''t'}"/>
<b text="{'Here be brackets ''}'' and ''{'''}"/>
</out>
</xsl:template>
</xsl:stylesheet>