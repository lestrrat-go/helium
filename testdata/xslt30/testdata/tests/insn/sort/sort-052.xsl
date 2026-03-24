<?spec xslt#sorting?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- tests sorting axes into document order  -->
  
  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:template match="div">
    <xsl:for-each select="br[following-sibling::node()]">
      <p class="seq1">
        <xsl:copy-of select="following-sibling::node()"/>
      </p>
      <p class="seq2">
        <xsl:copy-of select="following-sibling::br[1]/preceding-sibling::node()"/>
      </p>
      <p class="intersect">
        <xsl:copy-of
          select="following-sibling::node() intersect following-sibling::br[1]/preceding-sibling::node()"
        />
      </p>
    </xsl:for-each>
  </xsl:template>

</xsl:stylesheet>
