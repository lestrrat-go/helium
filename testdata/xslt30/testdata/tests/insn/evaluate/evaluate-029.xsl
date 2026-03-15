<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:s="http://my.com/f" exclude-result-prefixes="s">

<!-- Test use of xsl:evaluate in a predicate. Was saxon042. -->

    <xsl:template match="/">
    <out>
        <xsl:variable name="gt0" select="'@qty * @price &gt; 0'"/>
        <xsl:variable name="gt15" select="'@qty * @price &gt; 15'"/>
        <xsl:variable name="gt1500" select="'@qty * @price &gt; 1500'"/>            
        <gt0><xsl:value-of select="every $s in //sale satisfies $s[s:eval($gt0,.)]"/></gt0>
        <gt15><xsl:value-of select="every $s in //sale satisfies $s[s:eval($gt15,.)]"/></gt15>
        <gt1500><xsl:value-of select="every $s in //sale satisfies $s[s:eval($gt1500,.)]"/></gt1500>              
    </out>
    </xsl:template>

    <xsl:function name="s:eval">
      <xsl:param name="exp"/>
      <xsl:param name="value" as="item()"/>
      <xsl:evaluate xpath="$exp" context-item="$value"/>
    </xsl:function>    
</xsl:stylesheet>

