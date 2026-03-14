<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:s="http://s.com/" exclude-result-prefixes="s">

<!-- PURPOSE:   Test use of various axes within a sequence of parentless elements -->
<!-- (this test was originally designed to check the implementation of axes in a Saxon TinyTree forest,
     i.e. a TinyTree with multiple root nodes.) -->

<xsl:template match="/">
  <xsl:variable name="x" as="node()*">
    <xsl:call-template name="make-elements"/>
  </xsl:variable>
  <do>
   <xsl:apply-templates select="$x, 
                               $x[self::aaa]/a[1], $x[self::aaa]/a[1]/@att-a, 
                               $x/ddd/aaa/a[1], $x/ddd/aaa/a[1]/@att-a" mode="m"/>
  </do>
</xsl:template>

<xsl:template name="make-elements" as="node()*">
  <aaa><a att-a='10'>1</a><a>2</a></aaa>
  <bbb><b att-b='20'>1</b><b>2</b></bbb>
  <ccc><c att-c='30'>1</c><c>2</c></ccc>
  <xsl:document><ddd><aaa><a att-a='10'>1</a><a>2</a></aaa></ddd></xsl:document>
  <xsl:document><eee><aaa><a att-a='10'>1</a><a>2</a></aaa></eee></xsl:document>  
</xsl:template>

<xsl:template match="*|@*|/" mode="m">
<node name="{name()}">
  <child><xsl:value-of select="child::node()/name()" separator=","/></child>
  <ancestor><xsl:value-of select="ancestor::node()/name()" separator=","/></ancestor>  
  <ancestor-or-self><xsl:value-of select="ancestor-or-self::node()/name()" separator=","/></ancestor-or-self>
  <following-sibling><xsl:value-of select="following-sibling::node()/name()" separator=","/></following-sibling>
  <preceding-sibling><xsl:value-of select="preceding-sibling::node()/name()" separator=","/></preceding-sibling>  
  <following><xsl:value-of select="following::node()/name()" separator=","/></following>
  <preceding><xsl:value-of select="preceding::node()/name()" separator=","/></preceding>          
  <descendant><xsl:value-of select="descendant::node()/name()" separator=","/></descendant>  
  <descendant-or-self><xsl:value-of select="descendant-or-self::node()/name()" separator=","/></descendant-or-self>
</node>
</xsl:template>



</xsl:stylesheet>
