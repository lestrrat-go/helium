<xsl:stylesheet  
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  version="2.0"
>

<?spec xslt#named-templates?>
<!-- BUG: Saxon 5.3.1/004 -->
<!-- DESCRIPTION: When a template calls itself recursively within an xsl:for-each instruction, tail recursion is
    invoked but doesn't work correctly. The effect is that the loop is obeyed only once and
    with the wrong current node. -->

<xsl:template match="/">
<out>
   <xsl:for-each select="TEI.2/text/body">
    <xsl:call-template name="maintoc">
     <xsl:with-param name="thislevel">0</xsl:with-param>
    </xsl:call-template>
   </xsl:for-each>
</out>
</xsl:template>

<xsl:template name="maintoc">
 <xsl:param name="thislevel"/>
   <xsl:for-each select="div0|div1|div2|div3|div4|div5|div6">
    * <xsl:value-of select ="name()"/>
     <xsl:call-template name="maintoc">
      <xsl:with-param name="thislevel">
        <xsl:value-of select="$thislevel+1"/>
      </xsl:with-param>
     </xsl:call-template>
   </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
