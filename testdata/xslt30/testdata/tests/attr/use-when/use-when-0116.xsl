<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when on (included) xsl:stylesheet element -->
  <?spec xslt#conditional-inclusion?>
  
  <xsl:include href="use-when-0116a.xsl" use-when="true()"/>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  

</xsl:stylesheet>