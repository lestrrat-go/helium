<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- XSLT 2.0: no context item for xsl:number -->

<xsl:template match="/">
  <out>
    <xsl:iterate select="//*">
      <xsl:on-completion>
        <xsl:number/>
      </xsl:on-completion>
    </xsl:iterate>
  </out>    
</xsl:template>


</xsl:stylesheet>
