<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION:  4.1 Node Set Functions. -->
<!-- PURPOSE:   Test of 'last()' against each axis. -->
<xsl:strip-space elements="*"/>
<xsl:template match="b3">
<out>
    <ancestor>
    <xsl:for-each select="ancestor::node()">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </ancestor>
    <ancestor-or-self>
    <xsl:for-each select="ancestor-or-self::node()">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </ancestor-or-self>
    <descendant>
    <xsl:for-each select="descendant::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </descendant>
    <descendant-or-self>
    <xsl:for-each select="descendant-or-self::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </descendant-or-self>
    <following>
    <xsl:for-each select="following::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </following>    
    <following-sibling>
    <xsl:for-each select="following-sibling::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </following-sibling>   
    <parent>
    <xsl:for-each select="parent::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </parent>    
    <preceding>
    <xsl:for-each select="preceding::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </preceding>    
    <preceding-sibling>
    <xsl:for-each select="preceding-sibling::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </preceding-sibling>  
    <self>
    <xsl:for-each select="self::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </self>    
    <attribute>
    <xsl:for-each select="attribute::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </attribute>
    <namespace>
    <xsl:for-each select="namespace::*">
    <xsl:value-of select="concat(position(), ' of ', last())"/>,
    </xsl:for-each>
    </namespace>        
</out>
</xsl:template>
</xsl:stylesheet>