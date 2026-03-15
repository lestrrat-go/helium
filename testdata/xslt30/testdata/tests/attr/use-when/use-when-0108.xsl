<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  extension-element-prefixes="saxon"
  xmlns:saxon="http://not.saxon.sf.net/">


  <!-- PURPOSE: test xsl:use-when masking an extension element that doesn't exist -->
  <?spec xslt#conditional-inclusion?>
  <?dependency saxon:assign?>
  
  <xsl:variable name="count-para" select="0" saxon:assignable="yes"/>
  
  <xsl:template match="/">
  <result>
    <xsl:apply-templates/>
  <count nr="{$count-para}"/>
  </result>    
  </xsl:template>
  
  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para">
    <saxon:assign name="count-para" select="$count-para + 1" xsl:use-when="element-available('saxon:assign')"/>
    <xsl:next-match/>
  </xsl:template>
  

</xsl:stylesheet>