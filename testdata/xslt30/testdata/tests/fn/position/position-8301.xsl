<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">

    <!-- root -->
    <xsl:template match="root">
        <out>
         <xsl:for-each select="element | element/element[starts-with(@name, 'S')]">
             <xsl:value-of select="
                     if (some $x in element/@name
                         satisfies starts-with($x, 'S') and position() != 1) then
                         ('1')
                     else
                         ('2')"/>
         </xsl:for-each>
        </out>
    </xsl:template>
</xsl:stylesheet>
