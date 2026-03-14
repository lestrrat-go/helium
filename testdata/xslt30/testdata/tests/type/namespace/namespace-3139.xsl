<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Set default namespace in outer, then specify namespace for inner; add prefixed decl. -->
  <!-- NOTE: Processor developers could legitimately disagree about where the "p2" name
     has to be in scope on the result, if anywhere. Since there are no LREs inside foo,
     there is no requirement to pick up and emit the p2 declaration.
     The spec doesn't address this point. -->

<xsl:template match = "/">
  <out xmlns="http://abc.com/">
    <xsl:element namespace="http://abc.com/" name="foo" xmlns:p2="http://barz.com/">
      <xsl:element name="yyy"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>