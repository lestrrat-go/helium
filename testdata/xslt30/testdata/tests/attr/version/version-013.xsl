<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:ext="http://example.com/nosuchclass"
                exclude-result-prefixes="ext"
                version="1.0">

<!-- TEST that a call to an extension function that cannot be loaded is
     not an error unless we try to execute it -->
     <?spec xslt#backwards?>

<xsl:template match="/">
  <xsl:variable name="vendor" select="system-property('xsl:vendor')"/>
  <testing>
  <xsl:if test="$vendor = 'hi mom'">
    <xsl:value-of select="ext:foo()"/>
  </xsl:if>
  </testing>
</xsl:template>

</xsl:stylesheet>
