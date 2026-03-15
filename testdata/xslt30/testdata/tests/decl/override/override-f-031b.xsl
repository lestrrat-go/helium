<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs f"
    xmlns:f="http://localhost/fn"
    expand-text="yes"
    version="3.0">
    
    <!-- identity of union types -->
    
    <xsl:import-schema>
        <xs:schema>
            <xs:simpleType name="u2">
                <xs:union  memberTypes="xs:double xs:decimal"/>
            </xs:simpleType>
        </xs:schema>
    </xsl:import-schema>
    
    <xsl:use-package name="http://london2016/override-f-030/package-1"
        package-version="0.0.1">
        <xsl:override>
            <xsl:function name="f:a" as="u2" visibility="public">
                <xsl:param name="x" as="u2"/>
                <xsl:sequence select="xsl:original($x)"/>
            </xsl:function>
        </xsl:override>
    </xsl:use-package>

    <xsl:template name="xsl:initial-template" visibility="public">
        <out>
            <a>{f:a(12e0) instance of xs:double}</a>
            <b>{f:a(12.0) instance of xs:decimal}</b>
            <c>{f:a(xs:untypedAtomic('12')) instance of xs:decimal}</c>
            <d>{f:b(12e0) instance of xs:double}</d>
            <e>{f:b(12.0) instance of xs:decimal}</e>
            <f>{f:b(xs:untypedAtomic('12')) instance of xs:decimal}</f>
        </out>
    </xsl:template>
    
</xsl:package>