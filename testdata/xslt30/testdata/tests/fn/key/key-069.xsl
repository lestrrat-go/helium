<?xml version="1.0" encoding="UTF-8"?>

<!-- test xsl:key with type=xs:dateTime -->


<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- test key matching on dateTime values -->
    <!-- Should select a, c, d -->

    <xsl:key name="i" match="item" use="xs:dateTime(.)"/>

    <xsl:template match="/">
        <out>
            <xsl:value-of select="key('i', xs:dateTime('2002-10-05T22:00:00-05:00'))/@id"
                separator=","/>
        </out>
    </xsl:template>
</xsl:stylesheet>
