<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Two xmlns declarations with namespace attrib (matches default) in xsl:element. -->
  <!-- NOTE: Processor developers could legitimately disagree about where the "p2" name
     has to be in scope on the result. It must be in effect for yyy, but could be for foo
     as well. The spec doesn't address this point. -->
  <!-- MHK: I disagree with the above. I don't see how one could justify outputting the barz.com
     namespace on the foo element. But that's what Xalan appears to do... -->

<xsl:template match = "/">
  <out>
    <xsl:element name="foo" namespace="http://test.com/" xmlns="http://test.com/" xmlns:p2="http://barz.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>