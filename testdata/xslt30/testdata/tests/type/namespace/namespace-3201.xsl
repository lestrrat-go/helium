<?xml version="1.0"?>
<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
   xmlns:err="http://www.error.com/">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Discretionary: name="element-name-not-QName" choice="pass-through" -->
  <!-- Purpose: Test for error recovery when assigned name ends with : (has null local-part) -->
  <!-- MHK: changed to make it a dynamic error -->

<xsl:template match = "doc">
  <xsl:param name="c" select="':'"/>
  <out>
    <xsl:element name="{concat('err', $c)}">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>