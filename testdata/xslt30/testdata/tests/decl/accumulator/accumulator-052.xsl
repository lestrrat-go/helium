<xsl:stylesheet version="3.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">
    
    <!-- An accumulator whose initial value depends on a parameter. The idea here is to check that different
         transformations running this stylesheet against the same source document, perhaps concurrently, can have different
         values for the same accumulator. We can't do parallel running with the XSLT 3.0 test suite, but we can at least
         attempt serial execution -->
    
    <xsl:param name="start" as="xs:integer" required="true"/>

    <xsl:accumulator name="time" initial-value="$start">
        <xsl:accumulator-rule match="/" phase="end"   select="$value + 33"/>
    </xsl:accumulator>

    <xsl:mode use-accumulators="time"/>
    

    <xsl:template match="/">
        <root>
            <xsl:apply-templates select="*"/>
            <end value="{accumulator-after('time')}"/>
        </root>
    </xsl:template>
    
    <xsl:template match="*"/>

</xsl:stylesheet>
