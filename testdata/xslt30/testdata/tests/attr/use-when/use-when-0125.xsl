<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when expression with a non-applicable default namespace for elements and types. -->
  <?spec xslt#conditional-inclusion?>
  <?error XPST0003?>

  <xsl:template match="*" xpath-default-namespace="http://www.w3.org/2001/XMLSchema">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="*[local-name()='para']" use-when="'abc' instance of string">
    <p><xsl:next-match/></p>
  </xsl:template>

</xsl:stylesheet>