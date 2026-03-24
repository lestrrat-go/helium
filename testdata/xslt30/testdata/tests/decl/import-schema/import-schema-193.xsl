<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="my:my"
exclude-result-prefixes="xs my">
<xsl:output method="xml"/>
<xsl:import-schema>
    <xs:schema xsl:validation="strict">
        <xs:simpleType name="numeric">
            <xs:union memberTypes="xs:decimal xs:integer xs:float xs:double"/>
        </xs:simpleType>
    </xs:schema>
</xsl:import-schema>
<xsl:template name="main">
    <out><xsl:sequence select="my:add(3.3, 5.787)"/></out>
</xsl:template>

<xsl:function name="my:add" as="numeric">
    <xsl:param name="pLeft" as="numeric"/>
    <xsl:param name="pRight" as="numeric"/>
    <xsl:sequence select="($pLeft + $pRight)"/>
</xsl:function>
</xsl:stylesheet>