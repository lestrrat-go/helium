<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:key name="mykey" match="div" use="title"/>
<xsl:template match="doc">
<out>
      <!--Should say "template 1->Intro Section.": -->
      <P><xsl:apply-templates select="key('mykey', 'Introduction')"/></P>
      <!--Should say "template 2->SS Section.":-->
      <P><xsl:apply-templates select="key('mykey', 'Stylesheet Structure')/p"/></P>
      <!--Should say "template 3->Exp Section.":-->
      <P><xsl:apply-templates select="key('mykey', 'Expressions')/p"/></P>
      <!--Should say "template 4->Pat Section.":-->
      <P><xsl:apply-templates select="key('mykey', 'Patterns')/p"/></P>
</out>
</xsl:template>
  
<xsl:template match="key('mykey', 'Introduction')">
    <t1><xsl:value-of select="p"/></t1>
</xsl:template>
    
<xsl:template match="key('mykey', 'Stylesheet Structure')/p">
    <t2><xsl:value-of select="."/></t2>
</xsl:template>

<xsl:template match="key('mykey', 'Expressions')/p">
    <t3><xsl:value-of select="."/></t3>
</xsl:template>

<xsl:template match="key('mykey', 'Patterns')/p">
    <t4><xsl:value-of select="."/></t4>
</xsl:template>
    
</xsl:stylesheet>