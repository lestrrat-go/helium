<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#import?>
    <!-- Purpose: With two imports, precedence goes to the last one.
    Import precedence for templates matching foo (f imports g) is
    (high) h, f, g (low) -->

<xsl:import href="f.xsl"/>
<xsl:import href="h.xsl"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="foo"/>
  </out>
</xsl:template>

</xsl:stylesheet>