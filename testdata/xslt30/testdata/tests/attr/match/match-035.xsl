<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
  <!-- Purpose: Use xpath-default-namespace applied to a match attribute on the same element.
       xpath-default-namespace sets the default namespace for elements and types, but not for attributes -->
  
<xsl:output method="xml"/>

  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>
  
  <xsl:template match="*">
    <xsl:copy>
      <xsl:apply-templates select="@*"/>
    </xsl:copy>
  </xsl:template>

 <xsl:template match="@*">
  <xsl:copy/>
 </xsl:template>

 <!-- template does not match anything -->
 <xsl:template match="@schemaLocation"
  xpath-default-namespace="http://www.w3.org/2001/XMLSchema-instance">
  <xsl:message>FAILED!</xsl:message>
 </xsl:template>
  

  
</xsl:stylesheet>