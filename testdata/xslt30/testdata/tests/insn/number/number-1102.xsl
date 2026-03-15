<?xml version="1.0" encoding="iso-8859-1"?>
<?spec xslt#number?>
<!-- Test xsl:number level="single" applied to attribute nodes -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  version="2.0">


  <xsl:template match="/">
    <out>
      <!--<no-from>
        <xsl:for-each select="//@*">
          <n att="{name()}">
            <xsl:number/>
          </n>
        </xsl:for-each>
      </no-from>
      <from>
        <xsl:for-each select="//@*">
          <n att="{name()}">
            <xsl:number from="/"/>
          </n>
        </xsl:for-each>
      </from>
      <matching-count>
        <xsl:for-each select="//@*">
          <n att="{name()}">
            <xsl:number count="@*"/>
          </n>
        </xsl:for-each>
      </matching-count>-->
      <non-matching-count>
        <xsl:for-each select="//@*">
          <n att="{name()}">
            <xsl:number count="banana"/>
          </n>
        </xsl:for-each>
      </non-matching-count>
    </out>
  </xsl:template>

</xsl:stylesheet>
