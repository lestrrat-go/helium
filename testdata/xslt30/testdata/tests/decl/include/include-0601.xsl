<?xml version="1.0" encoding="iso-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                 version="2.0">

<xsl:import href="include-0601a.xsl"/>

<xsl:output method="xml"/>

<xsl:template match="text()">!<xsl:apply-imports/>!</xsl:template>

</xsl:stylesheet>
