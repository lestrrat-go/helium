<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: Test xsl:sort with sort parameters given as AVTs, 
     changing the values on successive runs -->

<xsl:template match="/">
   <out>
   <xsl:call-template name="sort">
    <xsl:with-param name="order" select="'ascending'"/>
    <xsl:with-param name="data-type" select="'text'"/>
   </xsl:call-template>
   <xsl:call-template name="sort">
    <xsl:with-param name="order" select="'descending'"/>
    <xsl:with-param name="data-type" select="'number'"/>
   </xsl:call-template>    
   <xsl:call-template name="sort">
    <xsl:with-param name="order" select="'ascending'"/>
    <xsl:with-param name="data-type" select="'number'"/>
   </xsl:call-template>    
   <xsl:call-template name="sort">
    <xsl:with-param name="order" select="'descending'"/>
    <xsl:with-param name="data-type" select="'text'"/>
   </xsl:call-template>
   </out>
</xsl:template>    
<xsl:template name="sort">
    <xsl:param name="order" select="'ascending'"/>
    <xsl:param name="data-type" select="'text'"/>
    <run>
      <xsl:for-each select="/doc/num[number(.) = number(.)]"> <!-- exclude the NaN values -->
        <xsl:sort data-type="{$data-type}" order="{$order}"/>
        <o><xsl:value-of select="."/></o>;
      </xsl:for-each>
    </run>

</xsl:template>
</xsl:stylesheet>
