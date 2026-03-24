<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:strip-space elements="*"/>

    
    <xsl:mode name="s" streamable="true"/>
    
    <xsl:template name="main" >
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:apply-templates mode="s"/>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:template match="*" mode="s">
      <xsl:copy>
        <xsl:copy-of select="@*"/>
        <xsl:attribute name="has-children" select="has-children(.)"/>
        <xsl:apply-templates mode="s"/>
      </xsl:copy>
    </xsl:template>        
    
</xsl:stylesheet>