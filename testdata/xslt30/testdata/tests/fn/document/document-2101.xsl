<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Test / as second argument to document().
      Two arguments: string variable, node-set. -->

  <xsl:variable name="typefile" select="'doc10.xml'"/>

  <xsl:template match="defaultcontent">
    <out>
      <xsl:copy-of select="document($typefile, /)"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
