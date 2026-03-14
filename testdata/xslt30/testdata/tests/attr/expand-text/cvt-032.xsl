<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" version="3.0">
    <xsl:template name="main" expand-text="yes">
        <xsl:variable name="stylesheet.attributes" as="attribute()*">
            <xsl:attribute name="xpath-default-namespace">http://www.w3.org/2000/svg</xsl:attribute>
        </xsl:variable>
        <foo>
            <xsl:sequence select="$stylesheet.attributes"/>
        </foo>
    </xsl:template>
</xsl:stylesheet>
