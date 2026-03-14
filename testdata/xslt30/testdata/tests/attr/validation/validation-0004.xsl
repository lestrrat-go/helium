<xsl:transform
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:ged="http://www.wrox.com/569090/gedcom"
   exclude-result-prefixes="xs ged"                          
   version="2.0" >
   
<!-- Demonstrates strict validation of attributes against a global attribute declaration (validation fails) -->

<!-- Erratum E10: validation of an attribute may cause a failure even if the attribute is subsequently 
      	overwritten by a valid attribute. -->

<xsl:strip-space elements="*"/>   

<xsl:import-schema schema-location="validation-0004.xsd"/>

<xsl:param name="attName" select="'Gender'"/>

<xsl:template name="main">
    <IndividualRec Id="ABCD" xsl:validation="strict">
        <xsl:for-each select="('A', 'B', 'C', 'F')">
            <xsl:attribute name="{$attName}" select="."/>
        </xsl:for-each>
    </IndividualRec>
</xsl:template>

</xsl:transform>
