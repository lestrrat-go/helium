<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<xsl:template name="main">
  <out>
  <xsl:for-each select="doc('../docs/works.xml')">
     <xsl:for-each select="snapshot(works/department/employee/@name)[position() = (1 to 3)]">
        <emp name="{.}" gender="{../@gender}" dept="{../../@name}"/>
     </xsl:for-each>
  </xsl:for-each>
  </out>
</xsl:template>


</xsl:stylesheet>