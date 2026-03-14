<xsl:transform
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:ged="http://www.wrox.com/569090/gedcom"
   xmlns="http://www.w3.org/1999/xhtml"
   exclude-result-prefixes="xs ged"                          
   version="2.0" >

<!-- show that whitespace stripping is not allowed on an element with simple content -->
<?spec xslt#strip?>

<xsl:strip-space elements="Religion"/>

<!-- import the schema for the GEDCOM 6.0 vocabulary -->

<xsl:import-schema schema-location="gedSchema.xsd"/>

<xsl:template match="/">
  <a c="{count(//*)}">x</a>
</xsl:template>

</xsl:transform>

