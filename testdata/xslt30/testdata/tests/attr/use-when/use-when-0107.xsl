<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test xsl:use-when on xsl:import-schema -->
  
  <xsl:import-schema namespace="foo" schema-location="foo.xsd" use-when="element-available('xsl:fried-fish')"/>
  <?spec xslt#conditional-inclusion?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  

</xsl:stylesheet>