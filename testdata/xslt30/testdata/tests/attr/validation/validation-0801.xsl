<xsl:transform
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://ns.example.com/val08/"
   exclude-result-prefixes="xs t"
   version="2.0" >

<?spec xpath#id-schema-element-test?>
<?spec xpath#id-element-test?>

<xsl:import-schema namespace="http://ns.example.com/val08/" schema-location="validation-08.xsd"/>

<xsl:template match="/">
    <out>
        <p1 a="{(//t:example)[1] instance of schema-element(t:block-level)}"/>
        <p2 a="{(//t:b)[1] instance of element(*, t:inline)}"/>
    </out>

</xsl:template>

</xsl:transform>

