<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- Purpose: Test an "xsl:when" instruction with and "xsl-if" instruction as part of its content template (if inside of when). -->
  <!-- SpecCitation: Rec="XSLT" version="1.0" type="" place="" -->
  <!-- Scenario: Operation="Standard-XML"-->
  

<xsl:template match="doc">
  <out>
    <xsl:choose>
      <xsl:when test = "2 > 1">
        <xsl:if test = "9 mod 3 = 0">
          Test executed successfully!!
        </xsl:if>
      </xsl:when>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
