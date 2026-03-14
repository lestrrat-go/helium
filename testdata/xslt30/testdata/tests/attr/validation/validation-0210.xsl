<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="http://ns.example.com/0210/"
    exclude-result-prefixes="xs t" input-type-annotations="preserve" version="2.0">

    <!-- test impact of input-type-annotations: inconsistent across modules -->

    <xsl:import href="validation-0210a.xsl"/>

    <xsl:import-schema namespace="http://ns.example.com/0210/" schema-location="validation-10.xsd"/>

    <xsl:template match="/">
        <out>
            <p1 a="{(//t:example)[1] instance of schema-element(t:block-level)}"/>
            <p2 a="{(//t:b)[1] instance of element(*, t:inline)}"/>
        </out>

    </xsl:template>

</xsl:transform>
