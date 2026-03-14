<?spec xslt#sorting?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- test sorting into document order  -->

  <xsl:strip-space elements="*"/>

  <xsl:template match="div">
    <out>
      <xsl:for-each select="br[following-sibling::node()]">
        <xsl:variable name="seq1" select="following-sibling::node()" as="node()*"/>
        <xsl:variable name="seq2" select="following-sibling::br[1]/preceding-sibling::node()"
          as="node()*"/>
        <xsl:variable name="intersect" select="$seq1 intersect $seq2" as="node()*"/>
        <p class="seq1">
          <xsl:copy-of select="$seq1"/>
        </p>
        <p class="seq2">
          <xsl:copy-of select="$seq2"/>
        </p>
        <p class="intersect">
          <xsl:copy-of select="$intersect"/>
        </p>
      </xsl:for-each>
    </out>
  </xsl:template>


</xsl:stylesheet>
