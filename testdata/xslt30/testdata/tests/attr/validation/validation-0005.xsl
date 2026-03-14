<xsl:transform
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   exclude-result-prefixes="xs"                          
   version="2.0" >
   
   <!-- test uniqueness constraints are applied when validating at element level (Erratum E12) -->
   
   <?spec xslt#stylesheets-and-schemas?>
<xsl:strip-space elements="*"/>   

<xsl:import-schema schema-location="gedSchema.xsd"/>

<xsl:param name="attName" select="'Gender'"/>

<xsl:template name="main">
    <GEDCOM xsl:validation="strict">
        <xsl:for-each select="1 to 3">
            <IndividualRec Id="ABCD">
                     <xsl:attribute name="{$attName}" select="'F'"/>
            </IndividualRec>
        </xsl:for-each>
    </GEDCOM>
</xsl:template>

</xsl:transform>
