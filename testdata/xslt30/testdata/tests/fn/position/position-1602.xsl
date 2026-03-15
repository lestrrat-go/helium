<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION:  4.1 Node Set Functions. -->
<!-- PURPOSE:   Test of '[position()=last()]' against each axis. -->
<xsl:strip-space elements="*"/>
<xsl:template match="b3">
<out>
    <ancestor>
    <xsl:for-each select="ancestor::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </ancestor>
    <ancestor-or-self>
    <xsl:for-each select="ancestor-or-self::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </ancestor-or-self>
    <descendant>
    <xsl:for-each select="descendant::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </descendant>
    <descendant-or-self>
    <xsl:for-each select="descendant-or-self::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </descendant-or-self>
    <following>
    <xsl:for-each select="following::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </following>    
    <following-sibling>
    <xsl:for-each select="following-sibling::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </following-sibling>   
    <parent>
    <xsl:for-each select="parent::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </parent>    
    <preceding>
    <xsl:for-each select="preceding::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </preceding>    
    <preceding-sibling>
    <xsl:for-each select="preceding-sibling::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </preceding-sibling>  
    <self>
    <xsl:for-each select="self::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </self>    
    <attribute>
    <!-- results depend on attribute order -->
    <xsl:for-each select="attribute::*[position()=last()]">
    <xsl:value-of select="name()"/>
    </xsl:for-each>
    </attribute>
</out>
</xsl:template>


</xsl:stylesheet>