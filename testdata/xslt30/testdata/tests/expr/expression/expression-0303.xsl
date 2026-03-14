<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-lang?>
<!-- PURPOSE:  Test of lang() function -->

<xsl:template match="doc">
<out>
<xsl:value-of select="para[@id='3' and lang('en')]"/>
</out>
</xsl:template>
</xsl:stylesheet>