<?xml version="1.0"?>
<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Test recovery when assigned name begins with : (has null namespace) -->
  <!-- MHK: changed to a run-time test -->
  <!-- MHK: error is now non-recoverable (April 2005) -->

<xsl:template match = "doc">
  <xsl:param name="prefix"/>
  <out>
    <xsl:element name="{$prefix}:foo">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>