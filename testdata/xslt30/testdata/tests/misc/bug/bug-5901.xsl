<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:d="urn:d"  exclude-result-prefixes="d">

  <xsl:strip-space elements="*"/>
  
  <?spec xslt#applying-templates?>
  
  <xsl:template match="/">
   <out><xsl:apply-templates/></out>
  </xsl:template>

  <xsl:template match="sources">
    <xsl:variable name="currentNode" select="."/>
    <xsl:copy>
      <xsl:apply-templates select="@*"/>
      <xsl:for-each select="document('xbug93cat.xml')/categories/cat">
        <xsl:variable name="nodesInCat" select="node"/>
        <xsl:if test="count($currentNode//pkg/*[local-name() = $nodesInCat/@n]) > 0" >
          <folder n="{@n}"/>
        </xsl:if>
      </xsl:for-each>
    </xsl:copy>
  </xsl:template>

</xsl:stylesheet>