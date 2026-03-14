<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:s="http://s.com/" exclude-result-prefixes="s">

  <!-- PURPOSE: Test xsl:perform-sort with different case-order options  -->

  <xsl:function name="s:en-uf">
    <xsl:param name="seq"/>
    <xsl:perform-sort select="$seq">
      <xsl:sort select="." data-type="text" lang="en-GB" case-order="upper-first"/>
    </xsl:perform-sort>
  </xsl:function>

  <xsl:function name="s:en-lf">
    <xsl:param name="seq"/>
    <xsl:perform-sort select="$seq">
      <xsl:sort select="." data-type="text" lang="en-GB" case-order="lower-first"/>
    </xsl:perform-sort>
  </xsl:function>

  <xsl:function name="s:en-def">
    <xsl:param name="seq"/>
    <xsl:perform-sort select="$seq">
      <xsl:sort select="." data-type="text" lang="en-GB"/>
    </xsl:perform-sort>
  </xsl:function>

  <xsl:template match="/">
    <result>
      <out lang="en-lower-first">
        <xsl:value-of select="s:en-lf(//item)" separator=","/>
      </out>
      <out lang="en-upper-first">
        <xsl:value-of select="s:en-uf(//item)" separator=","/>
      </out>
      <!-- don't test the default because it's implementation-defined -->
    </result>
  </xsl:template>
</xsl:stylesheet>
