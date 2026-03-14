<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:s="http://s.com/" exclude-result-prefixes="s">

  <!--  PURPOSE: Test xsl:sort-key with different languages.
     Note, the sort keys are chosen so that the results are predictable in any language,
     this is really just testing that the lang attribute is accepted.  -->

  <xsl:function name="s:en">
    <xsl:param name="p"/>
    <xsl:perform-sort select="$p">
      <xsl:sort select="." data-type="text" lang="en-GB"/>
    </xsl:perform-sort>
  </xsl:function>

  <xsl:function name="s:de">
    <xsl:param name="p"/>
    <xsl:perform-sort select="$p">
      <xsl:sort select="." data-type="text" lang="de-DE"/>
    </xsl:perform-sort>
  </xsl:function>

  <xsl:function name="s:sv">
    <xsl:param name="p"/>
    <xsl:perform-sort select="$p">
      <xsl:sort select="." data-type="text" lang="sv"/>
    </xsl:perform-sort>
  </xsl:function>

  <xsl:template match="/">
    <result>
      <out lang="en">
        <xsl:value-of select="s:en(//item)" separator=","/>
      </out>
      <out lang="de">
        <xsl:value-of select="s:de(//item)" separator=","/>
      </out>
      <out lang="sv">
        <xsl:value-of select="s:sv(//item)" separator=","/>
      </out>
    </result>
  </xsl:template>
</xsl:stylesheet>
