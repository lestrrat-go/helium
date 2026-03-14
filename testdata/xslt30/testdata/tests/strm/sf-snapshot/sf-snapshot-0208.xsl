<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<xsl:template name="main">
  <out>
  <xsl:source-document streamable="yes" href="../docs/works.xml">
     <xsl:for-each select="snapshot(remove(works/department/employee[1]/empnum/text(), 2))">
        <emp num="{.}" name="{../../@name}" gender="{../../@gender}" dept="{../../../@name}"/>
     </xsl:for-each>
  </xsl:source-document>
  </out>
</xsl:template>


</xsl:stylesheet>