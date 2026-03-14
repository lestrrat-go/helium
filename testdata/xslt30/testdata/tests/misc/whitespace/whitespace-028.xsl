<xsl:stylesheet version="    2.0 "
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- liberal use of whitespace in stylesheet attributes  -->
        <?spec xslt#strip?>

  <xsl:output method="
    &#x20;xml&#x20;
    " indent="
    &#x0a;no&#x09;" />

  <xsl:strip-space elements="  document "/>


  <xsl:template match="@*&#x09;|&#x09;node()" default-collation=" http://random/collation &#x9; http://www.w3.org/2005/xpath-functions/collation/codepoint  ">
    <xsl:element name="  {  concat(' ', name(), ' ')  }  ">
      <xsl:copy-of select=" @ * "  copy-namespaces="  yes  "/> 
      <xsl:apply-templates select="@*&#x09;|&#x09;node()"/>
    </xsl:element>
  </xsl:template>
  

</xsl:stylesheet>
