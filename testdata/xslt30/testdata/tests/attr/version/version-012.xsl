<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:ext="http://example.com/nosuchclass"
                extension-element-prefixes="ext"
                exclude-result-prefixes="ext"
                version="1.0">

<?spec xslt#extension-instruction?>
<?spec xslt#fallback?>
<!-- TEST that absent extension elements don't cause an error
     (a) if they aren't instantiated
     (b) if they have an xsl:fallback
-->

<xsl:template match="/">
  <xsl:variable name="vendor" select="system-property('xsl:vendor')"/>
  <testing>
  <xsl:if test="$vendor = 'hi mom'">
    <ext:foo/>
    <next/>
  </xsl:if>
  <ext:foo><xsl:fallback>fallback</xsl:fallback></ext:foo>
  </testing>
</xsl:template>

</xsl:stylesheet>

