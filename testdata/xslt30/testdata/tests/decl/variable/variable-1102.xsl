<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#variable-values?>
  <!-- Purpose: Test for handling of RTF used as positional index by setting
        variable as the content of the xsl:variable element.  Reference as [position()=$n]
        This is the case that will NOT obtain what the naive person expects. -->
  <!-- Author: Paul Dick -->
  <!-- Note: When a variable is used to select nodes by position, 
             be careful not to do: 
             <xsl:variable name="n">2</xsl:variable>
                ...
             <xsl:value-of select="item[$n]"/>
             This will output the value of the first item element, 
             because the variable n will be bound to a result tree fragment, 
             not a number. Instead, do either 

             <xsl:variable name="n" select="2"/>
                ...
             <xsl:value-of select="item[$n]"/>
                or 
             <xsl:variable name="n">2</xsl:variable>
                ...
             <xsl:value-of select="item[position()=$n]"/>     -->

<xsl:template match="doc">
  <xsl:variable name="n">2</xsl:variable>
  <out>
    <xsl:value-of select="item[position()=$n]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
