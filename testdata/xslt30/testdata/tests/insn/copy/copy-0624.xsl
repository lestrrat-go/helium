<?xml version='1.0' encoding='UTF-8' ?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:f="http://local-functions/"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="f xs">

    <!-- test combinations of copy-namespaces and inherit-namespaces -->
    <!-- same as copy-0620, but using the namespace axis rather than in-scope-prefixes -->
    
    <xsl:import href="copy-0620.xsl"/>
    
    <xsl:function name="f:in-scope-namespaces" as="xs:string">
        <xsl:param name="e" as="element(*)"/>
        <xsl:value-of>
            <xsl:for-each select="$e/namespace::*[name() != 'xml']">
                <xsl:sort select="name()"/>
                <xsl:variable name="ok" select="if (.. is $e) then '' else '(BAD!)'"/>
                <xsl:value-of select="concat('|', name(), $ok, '=', string(.))"/>
            </xsl:for-each> 
        </xsl:value-of>       
    </xsl:function>

</xsl:stylesheet>
