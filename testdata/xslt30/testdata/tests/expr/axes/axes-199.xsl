<xsl:stylesheet
xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
version="2.0" >

<?spec xpath#axes?>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
   <xsl:template match="Text">
      <xsl:if test="not(@Object = preceding::Text/@Object)">
        <xsl:copy>
          <xsl:apply-templates select="@*|node()"/>
        </xsl:copy>
      </xsl:if>
   </xsl:template>
   
    <xsl:template match="@*|node()">
      <xsl:copy>
          <xsl:apply-templates select="@*|node()"/>
      </xsl:copy>
    </xsl:template>
</xsl:stylesheet>
