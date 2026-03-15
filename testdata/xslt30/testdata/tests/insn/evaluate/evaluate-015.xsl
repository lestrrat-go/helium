<xsl:transform version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:eval="http://example.com/eval"
  exclude-result-prefixes="eval">

  <!-- Basic test of saxon:evaluate-node() -->
  <!-- Formerly saxon013 -->
  <!-- rewritten Nov 2009 to use xsl:evaluate -->

  <xsl:template match="doc">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:template match="exp">
    <ans>
      <xsl:evaluate xpath="string(.)" base-uri="{base-uri(.)}" namespace-context="."
        context-item="."/>
    </ans>
  </xsl:template>
  
  <xsl:strip-space elements="*"/>
  
  <xsl:function name="eval:node" visibility="final">
    <xsl:param name="node" as="element()" />
    
    <xsl:evaluate xpath="string($node)" base-uri="{base-uri($node)}" namespace-context="$node" />
      
  </xsl:function>

</xsl:transform>
