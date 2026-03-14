<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <xsl:template name="xsl:initial-template">
       <out>
        <xsl:source-document streamable="yes" href="si-iterate-035.xml">
            <xsl:iterate select="employees/employee">
                <xsl:param name="highest" as="element(employee)*"/>
                <xsl:param name="lowest" as="element(employee)*"/>
                <xsl:on-completion>
                    <highest-paid-employees>
                        <xsl:value-of select="$highest/name"/>
                    </highest-paid-employees>
                    <lowest-paid-employees>
                        <xsl:value-of select="$lowest/name"/>
                    </lowest-paid-employees>  
                </xsl:on-completion>
                <xsl:variable name="this" select="copy-of()"/>
                <xsl:variable name="is-new-highest" as="xs:boolean"
                    select="empty($highest[@salary ge current()/@salary])"/>
                <xsl:variable name="is-equal-highest" as="xs:boolean" 
                    select="exists($highest[@salary eq current()/@salary])"/> 
                <xsl:variable name="is-new-lowest" as="xs:boolean" 
                    select="empty($lowest[@salary le current()/@salary])"/>
                <xsl:variable name="is-equal-lowest" as="xs:boolean" 
                    select="exists($lowest[@salary eq current()/@salary])"/> 
                <xsl:variable name="new-highest-set" as="element(employee)*"
                    select="if ($is-new-highest) then $this
                    else if ($is-equal-highest) then ($highest, $this)
                    else $highest"/>
                <xsl:variable name="new-lowest-set" as="element(employee)*"
                    select="if ($is-new-lowest) then $this
                    else if ($is-equal-lowest) then ($lowest, $this)
                    else $lowest"/>
                <xsl:next-iteration>
                    <xsl:with-param name="highest" select="$new-highest-set"/>
                    <xsl:with-param name="lowest" select="$new-lowest-set"/>
                </xsl:next-iteration>
            </xsl:iterate>
        </xsl:source-document>
       </out>
    </xsl:template>
</xsl:stylesheet>