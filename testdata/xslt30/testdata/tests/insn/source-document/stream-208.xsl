<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all"
    expand-text="true"
    >
    
    
    <xsl:accumulator name="count-measures" as="xs:integer" initial-value="0" streamable="yes">
        <xsl:accumulator-rule match="measure" select="$value + 1"/>
    </xsl:accumulator>
    
    <xsl:template name="xsl:initial-template">
        <data>
            <xsl:source-document href="Psalm100.xml" streamable="yes" use-accumulators="count-measures">
                <has-measure>{exists(//measure)}</has-measure>
                <count-measures>{accumulator-after('count-measures')}</count-measures>
            </xsl:source-document>
        </data>     
    </xsl:template>

</xsl:stylesheet>